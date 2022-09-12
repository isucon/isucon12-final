using System;
using System.Threading.Tasks;
using Data;

namespace Network
{
    [Serializable]
    public class UpdateDeckRequest : CommonRequest
    {
        public long[] cardIds;
    }

    [Serializable]
    public class UpdateDeckResponse : CommonResponse
    {
    }
    
    public class UpdateDeckApi : ApiBase<UpdateDeckRequest, UpdateDeckResponse>
    {
        public override string Path => $"/user/{GameManager.userData.user.id}/card";

        public UpdateDeckApi(UpdateDeckRequest req)
        {
            RequestData = req;
        }
    }

    public partial class ApiClient
    {
        public async Task<UpdateDeckResponse> UpdateDeckAsync(long[] cardIds)
        {
            var req = new UpdateDeckRequest
            {
                cardIds = cardIds,
            };
            var api = new UpdateDeckApi(req);
            var res = await Post(api);
            return res;
        }
    }
}
