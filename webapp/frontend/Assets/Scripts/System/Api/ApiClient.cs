using System;
using System.Net;
using System.Text;
using System.Threading.Tasks;
using UnityEngine;
using UnityEngine.Networking;

namespace Network
{
    public partial class ApiClient
    {
        private string _host = "http://localhost:8080";

        public string Host
        {
            get
            {
                return _host;
            }
            set
            {
                if (!string.IsNullOrEmpty(value))
                {
                    _host = value;
                }
            }
        }

        public string ViewerId { get; private set; }
        
        public string SessionId { get; private set; }

        private string _masterVersion = "1";

        private string _oneTimeToken;
        
        public ApiClient(string host)
        {
            Host = host;
        }

        private async Task<TResponse> Post<TRequest, TResponse>(ApiBase<TRequest, TResponse> api)
            where TRequest : CommonRequest
            where TResponse : CommonResponse
        {
            var req = await PostOnce(api);
            if (req.responseCode == (long)HttpStatusCode.UnprocessableEntity)
            {
                var error = JsonUtility.FromJson<ErrorResponse>(req.downloadHandler.text);
                if (error.message == "invalid master version")
                {
                    // マスターバージョン不一致なので、1と2をスイッチして再度リクエストする
                    _masterVersion = _masterVersion == "1" ? "2" : "1";
                    Debug.Log("Switch master version: " + _masterVersion);
                    req.Dispose();
                    req = await PostOnce(api);
                }
            }

            var statusCode = (int)req.responseCode;
            Debug.Log("POST status: " + statusCode);
            if (req.result != UnityWebRequest.Result.Success)
            {
                var httpError = req.error;
                var error = req.downloadHandler.text;
                Debug.Log("POST error: " + httpError);
                req.Dispose();
                DialogManager.Instance.ShowMessageDialog("APIエラー", error);
                throw new ApiException(statusCode, error);
            }

            var resText = req.downloadHandler.text;
            Debug.Log("API response: " + resText);
            var response = JsonUtility.FromJson<TResponse>(resText);
            req.Dispose();

            if (response.updatedResources?.user != null && response.updatedResources.user.id != 0)
            {
                GameManager.userData.user.isuCoin = response.updatedResources.user.isuCoin;
                GameManager.userData.isuCoin.refreshTime = response.updatedResources.user.lastGetRewardAt;
            }

            return response;
        }

        private async Task<UnityWebRequest> PostOnce<TRequest, TResponse>(ApiBase<TRequest, TResponse> api)
            where TRequest : CommonRequest
            where TResponse : CommonResponse
        {
            var url = _host + api.Path;
            Debug.Log("POST url: " + url);
            var req = new UnityWebRequest(url);
            
            req.method = api.Method;
            req.SetRequestHeader("x-master-version", _masterVersion);
            if (SessionId != null)
            {
                req.SetRequestHeader("x-session", SessionId);
            }
            req.downloadHandler = new DownloadHandlerBuffer();
            
            if (api.RequestData != null)
            {
                if (string.IsNullOrEmpty(api.RequestData.viewerId))
                {
                    api.RequestData.viewerId = ViewerId;
                }
                
                req.SetRequestHeader("Content-Type", "application/json");
                var postData = JsonUtility.ToJson(api.RequestData);
                Debug.Log("post data: " + postData);
                req.uploadHandler = new UploadHandlerRaw(Encoding.UTF8.GetBytes(postData));
            }

            await req.SendWebRequest();
            return req;
        }
    }

    public class ApiException : Exception
    {
        public int StatusCode { get; set; }

        public ApiException(int statusCode, string message) : base(message)
        {
            StatusCode = statusCode;
        }

        public override string ToString()
        {
            return $"ApiException: Status={StatusCode}, Message='{Message}'";
        }
    }
}
